package model

// SlideLayout describes the visual arrangement pattern of a slide.
type SlideLayout string

const (
	// LayoutTitle only has a centered title, optionally a subtitle.
	LayoutTitle SlideLayout = "title"

	// LayoutTitleContent has a title at top, content area below.
	LayoutTitleContent SlideLayout = "title_content"

	// LayoutTwoColumn splits content into left/right halves.
	LayoutTwoColumn SlideLayout = "two_column"

	// LayoutImageLeft has an image on the left, text on the right.
	LayoutImageLeft SlideLayout = "image_left"

	// LayoutImageRight has an image on the right, text on the left.
	LayoutImageRight SlideLayout = "image_right"

	// LayoutImageFull is a full-bleed background image with overlay text.
	LayoutImageFull SlideLayout = "image_full"

	// LayoutSection is a section divider slide.
	LayoutSection SlideLayout = "section"
)
