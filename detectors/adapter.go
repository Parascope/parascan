package detectors

// SimpleDetectorAdapter adapts SimpleDetector to Detector interface
type SimpleDetectorAdapter struct {
	simple SimpleDetector
}

func NewSimpleDetectorAdapter(simple SimpleDetector) *SimpleDetectorAdapter {
	return &SimpleDetectorAdapter{
		simple: simple,
	}
}

func (a *SimpleDetectorAdapter) Name() string {
	return a.simple.Name()
}

func (a *SimpleDetectorAdapter) Detect(ctx *DetectionContext) (map[string]string, error) {
	return a.simple.Detect(ctx.ProjectPath)
}