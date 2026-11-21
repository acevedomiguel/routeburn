package ui

// Class describes a terminal size bucket.
type Class int

// Layout classes, ordered from smallest to largest.
const (
	ClassTooSmall Class = iota
	ClassNarrow
	ClassNormal
	ClassWide
)

// MinWidth and MinHeight are the smallest usable terminal dimensions.
const (
	MinWidth  = 60
	MinHeight = 16
)

// TooSmallMessage is shown when the terminal cannot fit the dashboard.
const TooSmallMessage = "Terminal too small; resize to at least 60x16"

// Layout is the resolved geometry for a terminal size.
type Layout struct {
	Class        Class
	Width        int
	Height       int
	ContentWidth int
	Padding      int
	TwoColumn    bool
	Scrollable   bool
	Message      string
}

// Classify maps a terminal size to a layout class and content width.
func Classify(width, height int) Layout {
	l := Layout{Width: width, Height: height}
	if width < MinWidth || height < MinHeight {
		l.Class = ClassTooSmall
		l.Message = TooSmallMessage
		if width > 0 {
			l.ContentWidth = width
		}
		return l
	}

	switch {
	case width >= 100:
		l.Class = ClassWide
		l.Padding = 2
	case width >= 80:
		l.Class = ClassNormal
		l.Padding = 1
	default:
		l.Class = ClassNarrow
		l.Padding = 1
		l.Scrollable = true
	}

	l.ContentWidth = width - 2*l.Padding
	if l.ContentWidth < 1 {
		l.ContentWidth = width
	}
	l.TwoColumn = l.Class == ClassWide
	return l
}
