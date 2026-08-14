package model

// ElementType describes what kind of content an element holds.
type ElementType string

const (
	ElementTitle    ElementType = "title"
	ElementSubtitle ElementType = "subtitle"
	ElementBody     ElementType = "body"
	ElementBullet   ElementType = "bullet"
	ElementImage    ElementType = "image"
	ElementShape    ElementType = "shape"
	ElementTable    ElementType = "table"
	ElementChart    ElementType = "chart"
	ElementConnector ElementType = "connector"
	ElementIcon     ElementType = "icon"
	ElementGroup    ElementType = "group"
)

// ShapeType enumerates supported auto shapes.
type ShapeType string

const (
	ShapeRectangle        ShapeType = "rectangle"
	ShapeRoundedRectangle ShapeType = "rounded_rectangle"
	ShapeEllipse          ShapeType = "ellipse"
	ShapeTriangle         ShapeType = "triangle"
	ShapeDiamond          ShapeType = "diamond"
	ShapeLine             ShapeType = "line"
	ShapeArrow            ShapeType = "arrow"
	ShapeDoubleArrow      ShapeType = "double_arrow"
	ShapePentagon         ShapeType = "pentagon"
	ShapeHexagon          ShapeType = "hexagon"
	ShapeStar             ShapeType = "star"
	ShapeCallout          ShapeType = "callout"
	ShapeHeart            ShapeType = "heart"
	ShapeCloud            ShapeType = "cloud"
)

// ChartType enumerates supported chart types.
type ChartType string

const (
	ChartBar       ChartType = "bar"
	ChartColumn    ChartType = "column"
	ChartLine      ChartType = "line"
	ChartPie       ChartType = "pie"
	ChartArea      ChartType = "area"
	ChartScatter   ChartType = "scatter"
	ChartDoughnut  ChartType = "doughnut"
	ChartCombo     ChartType = "combo" // mixed bar+line with optional secondary axis
	ChartBar3D     ChartType = "bar_3d"
	ChartColumn3D  ChartType = "column_3d"
	ChartLine3D    ChartType = "line_3d"
	ChartPie3D     ChartType = "pie_3d"
	ChartArea3D    ChartType = "area_3d"
)

// BackgroundType enumerates background fill types.
type BackgroundType string

const (
	BgSolid    BackgroundType = "solid"
	BgGradient BackgroundType = "gradient"
	BgImage    BackgroundType = "image"
)

// GradientType for gradient backgrounds.
type GradientType string

const (
	GradientLinear  GradientType = "linear"
	GradientRadial  GradientType = "radial"
)

// TransitionType for slide transitions.
type TransitionType string

const (
	TransitionNone   TransitionType = "none"
	TransitionFade   TransitionType = "fade"
	TransitionPush   TransitionType = "push"
	TransitionWipe   TransitionType = "wipe"
	TransitionSplit  TransitionType = "split"
	TransitionCover  TransitionType = "cover"
	TransitionZoom   TransitionType = "zoom"
	TransitionMorph  TransitionType = "morph"
)

// AnimationType for element animations.
type AnimationType string

const (
	AnimFade      AnimationType = "fade"
	AnimFlyIn     AnimationType = "fly_in"
	AnimZoomIn    AnimationType = "zoom_in"
	AnimBounce    AnimationType = "bounce"
	AnimRotate    AnimationType = "rotate"
	AnimWipe      AnimationType = "wipe"
	AnimAppear    AnimationType = "appear"
)

// AnimationTrigger defines when an animation fires.
type AnimationTrigger string

const (
	TriggerOnClick     AnimationTrigger = "on_click"
	TriggerAfterPrev   AnimationTrigger = "after_previous"
	TriggerWithPrev    AnimationTrigger = "with_previous"
)

// AnimationDirection for directional animations.
type AnimationDirection string

const (
	DirFromLeft   AnimationDirection = "left"
	DirFromRight  AnimationDirection = "right"
	DirFromTop    AnimationDirection = "top"
	DirFromBottom AnimationDirection = "bottom"
	DirFromCenter AnimationDirection = "center"
)
