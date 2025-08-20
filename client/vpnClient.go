package client

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"math"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	"golang.org/x/image/draw"
)

func createSession(host, password string) (*resty.Client, error) {
	jar, _ := cookiejar.New(nil)
	client := resty.New().
		SetBaseURL(host).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetCookieJar(jar)

	resp, err := client.R().
		SetBody(map[string]string{"password": password}).
		Post("/api/session")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("login failed: %s - %s", resp.Status(), resp.String())
	}
	return client, nil
}

type WGClient struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// добавь нужные поля, если понадобятся
}

func filenameFromContentDisposition(cd string) string {
	// пробуем filename*, потом filename
	// RFC 5987: filename*=UTF-8''<urlencoded>
	reStar := regexp.MustCompile(`(?i)filename\*\s*=\s*UTF-8''([^;]+)`)
	if m := reStar.FindStringSubmatch(cd); len(m) == 2 {
		if dec, err := url.QueryUnescape(m[1]); err == nil && dec != "" {
			return dec
		}
	}
	re := regexp.MustCompile(`(?i)filename\s*=\s*"?([^";]+)"?`)
	if m := re.FindStringSubmatch(cd); len(m) == 2 {
		return m[1]
	}
	return ""
}

func downloadClientConfig(client *resty.Client, clientID, destPath string) (string, error) {
	endpoint := fmt.Sprintf("/api/wireguard/client/%s/configuration", url.PathEscape(clientID))

	resp, err := client.R().
		SetDoNotParseResponse(true). // получим io.ReadCloser
		Get(endpoint)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.RawBody().Close() }()

	if resp.IsError() {
		body, _ := io.ReadAll(resp.RawBody())
		return "", fmt.Errorf("download failed: %s - %s", resp.Status(), strings.TrimSpace(string(body)))
	}

	// определяем имя файла
	if destPath == "" {
		cd := resp.Header().Get("Content-Disposition")
		if fn := filenameFromContentDisposition(cd); fn != "" {
			destPath = fn
		} else {
			// запасной вариант
			destPath = clientID + ".conf"
		}
	}

	out, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.RawBody()); err != nil {
		return "", err
	}
	return destPath, nil
}

func getQrCode(client *resty.Client, clientID string) ([]byte, error) {
	endpoint := fmt.Sprintf("/api/wireguard/client/%s/qrcode.svg", clientID)

	resp, err := client.R().Get(endpoint)
	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		return nil, fmt.Errorf("getQrCode failed: %s - %s", resp.Status(), resp.String())
	}

	return resp.Body(), nil
}

func getQrCodeString(client *resty.Client, clientID string) (string, error) {
	endpoint := fmt.Sprintf("/api/wireguard/client/%s/qrcode.svg", clientID)

	resp, err := client.R().Get(endpoint)
	if err != nil {
		return "", err
	}

	if resp.IsError() {
		return "", fmt.Errorf("getQrCode failed: %s - %s", resp.Status(), resp.String())
	}

	return resp.String(), nil
}

// получить список клиентов
func listClients(client *resty.Client) ([]WGClient, error) {
	var out []WGClient
	resp, err := client.R().
		SetResult(&out).
		Get("/api/wireguard/client")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("list clients failed: %s - %s", resp.Status(), resp.String())
	}
	return out, nil
}

// найти clientId по имени (точное совпадение)
func findClientIDByName(client *resty.Client, name string) (string, error) {
	cls, err := listClients(client)
	if err != nil {
		return "", err
	}
	for _, c := range cls {
		if c.Name == name {
			return c.ID, nil
		}
	}
	return "", errors.New("client not found by name")
}

func createClient(client *resty.Client, name string) error {
	body := map[string]string{"name": name}

	resp, err := client.R().
		SetBody(body).
		Post("/api/wireguard/client")

	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("create client failed: %s - %s", resp.Status(), resp.String())
	}

	fmt.Println("Client created OK:", resp.String())
	return nil
}

func deleteClient(client *resty.Client, clientID string) error {
	endpoint := fmt.Sprintf("/api/wireguard/client/%s", clientID)

	resp, err := client.R().
		Delete(endpoint)
	if err != nil {
		return err
	}

	if resp.IsError() {
		return fmt.Errorf("delete client failed: %s - %s", resp.Status(), resp.String())
	}

	return nil
}

func DoCreateConfig(host, password, region, tgId, platform string) (string, []byte, error) {
	cli, err := createSession(host, password)
	if err != nil {
		return "", nil, err
	}

	clientConfig := fmt.Sprintf("%s-%s%s", tgId, region, platform)

	if err := createClient(cli, clientConfig); err != nil {
		return "", nil, err
	}

	clientID, err := findClientIDByName(cli, clientConfig)
	if err != nil {
		return "", nil, err
	}

	path, err := downloadClientConfig(cli, clientID, "")
	if err != nil {
		return "", nil, err
	}

	qrCode, err := getQrCodeString(cli, clientID)

	pngQr, err := svgToPng(qrCode, 512, 512)
	if err != nil {
		return "", nil, err
	}

	return path, pngQr, nil
}

var vbRe = regexp.MustCompile(`viewBox\s*=\s*"[^"]*\b0\s+0\s+(\d+)\s+(\d+)\b"`)

func svgToPNGForQR(svgStr string, target int) ([]byte, error) {
	if target <= 0 {
		target = 512
	}

	// вытащим исходную сетку из viewBox (у вашего SVG это 89x89)
	wUnits, hUnits := 89, 89
	if m := vbRe.FindStringSubmatch(svgStr); len(m) == 3 {
		if v, err := strconv.Atoi(m[1]); err == nil {
			wUnits = v
		}
		if v, err := strconv.Atoi(m[2]); err == nil {
			hUnits = v
		}
	}

	scale := int(math.Max(1, math.Round(float64(target)/float64(wUnits))))
	baseW := wUnits * scale
	baseH := hUnits * scale

	// рисуем в «базовый» размер (кратный сетке)
	icon, err := oksvg.ReadIconStream(strings.NewReader(svgStr))
	if err != nil {
		return nil, err
	}
	icon.SetTarget(0, 0, float64(baseW), float64(baseH))

	src := image.NewRGBA(image.Rect(0, 0, baseW, baseH))
	scanner := rasterx.NewScannerGV(baseW, baseH, src, src.Bounds())
	raster := rasterx.NewDasher(baseW, baseH, scanner)
	icon.Draw(raster, 1.0)

	// если нужен ровно 512 — ресайзим NearestNeighbor (сохраняет «квадраты»)
	dst := src
	if baseW != target || baseH != target {
		dst = image.NewRGBA(image.Rect(0, 0, target, target))
		draw.NearestNeighbor.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func svgToPng(svg string, width, height int) ([]byte, error) {
	cmd := exec.Command("rsvg-convert", "-f", "png", "-w", "512", "-h", "512")
	cmd.Stdin = bytes.NewReader([]byte(svg))
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	pngBytes := out.Bytes()
	return pngBytes, nil
}
