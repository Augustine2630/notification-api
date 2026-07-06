package bot

type Step int

const (
	StepIdle Step = iota
	StepChooseRegion
	StepChooseAction
	StepChoosePlatform
	StepEnterName
)

type UserState struct {
	Step     Step
	Region   string
	Platform string
}
