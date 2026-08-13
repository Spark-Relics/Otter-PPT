package model

// GradientStop is a single color stop in a gradient.
type GradientStop struct {
	Color    string  `json:"color"`             // hex
	Position float64 `json:"position"`          // 0-100
	Opacity  float64 `json:"opacity,omitempty"` // 0-1, 0 means opaque for compatibility
}

// Gradient defines a gradient fill.
type Gradient struct {
	Type  GradientType   `json:"type"`
	Angle float64        `json:"angle,omitempty"` // degrees, for linear
	Stops []GradientStop `json:"stops"`
}

// Background defines a slide or presentation background.
type Background struct {
	Type      BackgroundType `json:"type"`
	Color     string         `json:"color,omitempty"`      // for solid
	Gradient  *Gradient      `json:"gradient,omitempty"`   // for gradient
	ImagePath string         `json:"image_path,omitempty"` // for image
}

// Transition defines a slide transition effect.
type Transition struct {
	Type     TransitionType `json:"type"`
	Duration float64        `json:"duration,omitempty"` // seconds
}

// Animation defines an animation on an element.
type Animation struct {
	Type      AnimationType      `json:"type"`
	Trigger   AnimationTrigger   `json:"trigger"`
	Direction AnimationDirection `json:"direction,omitempty"`
	Duration  float64            `json:"duration,omitempty"` // seconds
	Delay     float64            `json:"delay,omitempty"`    // seconds
}
