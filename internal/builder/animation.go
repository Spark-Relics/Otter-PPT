package builder

import (
	"fmt"
	"strings"

	"github.com/otter-ppt/otter-ppt/internal/model"
)

// buildTiming generates the <p:timing> XML block for element animations.
// It creates a sequential main timeline where each animated element appears
// as a child <p:par> node inside the main sequence.
func (b *Builder) buildTiming(slide *model.Slide) string {
	var animated []*model.Element
	for _, e := range slide.Elements {
		if e.Animation != nil && e.Animation.Type != "" {
			animated = append(animated, e)
		}
	}
	if len(animated) == 0 {
		return ""
	}

	// Calculate cumulative delays (in ms) for after_previous / with_previous
	prevEnd := 0 // ms
	var children strings.Builder

	for i, elem := range animated {
		anim := elem.Animation
		objID := ooxmlObjectID(elem.ID)

		// Duration in ms
		durMs := int(anim.Duration * 1000)
		if durMs == 0 {
			durMs = 500
		}
		delayMs := int(anim.Delay * 1000)

		// Determine start time relative to previous
		var startMs int
		switch anim.Trigger {
		case model.TriggerAfterPrev:
			startMs = prevEnd + delayMs
		case model.TriggerWithPrev:
			if i == 0 {
				startMs = delayMs
			} else {
				startMs = prevEnd // overlap with previous start (approximation)
			}
		default: // on_click
			startMs = delayMs
		}

		children.WriteString(buildAnimParXML(objID, anim, startMs, durMs, i))

		// Update prevEnd — for "on_click" the next click resets, so keep prevEnd for after_previous logic
		if anim.Trigger != model.TriggerOnClick {
			prevEnd = startMs + durMs
		} else {
			prevEnd = startMs + durMs
		}
	}

	// Build the full timing tree
	return fmt.Sprintf(`<p:timing><p:tnLst>
<p:par><p:cTn id="1" dur="indefinite" restart="never" nodeType="tmRoot">
<p:childTnLst>
<p:seq concurrent="1" nextAc="seek"><p:cTn id="2" dur="indefinite" nodeType="mainSeq">
<p:childTnLst>%s</p:childTnLst>
</p:cTn>
<p:prevCondLst><p:cond evt="onPrev" delay="0"><p:tgtEl><p:sldTgt/></p:tgtEl></p:cond></p:prevCondLst>
<p:nextCondLst><p:cond evt="onNext" delay="0"><p:tgtEl><p:sldTgt/></p:tgtEl></p:cond></p:nextCondLst>
</p:seq>
</p:childTnLst>
</p:cTn></p:par>
</p:tnLst></p:timing>`, children.String())
}

// buildAnimParXML generates a single <p:par> animation node for one element.
func buildAnimParXML(objID uint32, anim *model.Animation, startMs, durMs, index int) string {
	// Build the effect XML based on animation type
	effect, presetClass, presetID := animationEffect(anim.Type, anim.Direction)

	// Node type for the condition
	nodeType := "clickEffect"
	if anim.Trigger == model.TriggerAfterPrev {
		nodeType = "afterEffect"
	} else if anim.Trigger == model.TriggerWithPrev {
		nodeType = "withEffect"
	}

	cTnID := index*4 + 3

	return fmt.Sprintf(`<p:par><p:cTn id="%d" fill="hold">
<p:stCondLst><p:cond delay="%d"/></p:stCondLst>
<p:childTnLst>
<p:par><p:cTn id="%d" fill="hold">
<p:stCondLst><p:cond delay="0"/></p:stCondLst>
<p:childTnLst>
<p:par><p:cTn id="%d" presetID="%d" presetClass="%s" presetSubtype="0" fill="hold" grpId="0" nodeType="%s">
<p:stCondLst><p:cond delay="0"/></p:stCondLst>
<p:childTnLst>
<p:set><p:cBhvr><p:cTn id="%d" dur="1" fill="hold"><p:stCondLst><p:cond delay="0"/></p:stCondLst></p:cTn>
<p:tgtEl><p:spTgt spid="%d"/></p:tgtEl>
<p:attrNameLst><p:attrName>style.visibility</p:attrName></p:attrNameLst>
</p:cBhvr>
<p:to><p:strVal val="visible"/></p:to>
</p:set>
<p:animEffect transition="in" filter="%s"><p:cBhvr><p:cTn id="%d" dur="%d"/><p:tgtEl><p:spTgt spid="%d"/></p:tgtEl></p:cBhvr></p:animEffect>
</p:childTnLst>
</p:cTn></p:par>
</p:childTnLst>
</p:cTn></p:par>
</p:childTnLst>
</p:cTn></p:par>`,
		cTnID, startMs,
		cTnID+1,
		cTnID+2, presetID, presetClass, nodeType,
		cTnID+3,
		objID,
		effect,
		cTnID+4, durMs, objID)
}

// animationEffect maps AnimationType + Direction to OOXML filter / preset values.
// Returns: filter string, presetClass, presetID.
func animationEffect(animType model.AnimationType, direction model.AnimationDirection) (filter, presetClass string, presetID int) {
	presetClass = "entr"
	presetID = 1

	switch animType {
	case model.AnimFade:
		filter = "fade"
		presetID = 10
	case model.AnimFlyIn:
		presetID = 2
		switch direction {
		case model.DirFromLeft:
			filter = "wipe(left)"
		case model.DirFromRight:
			filter = "wipe(right)"
		case model.DirFromTop:
			filter = "wipe(up)"
		case model.DirFromBottom:
			filter = "wipe(down)"
		default:
			filter = "wipe(left)"
		}
	case model.AnimZoomIn:
		filter = "zoom"
		presetID = 23
	case model.AnimBounce:
		filter = "rise"
		presetID = 16
	case model.AnimRotate:
		filter = "spin"
		presetID = 53
	case model.AnimWipe:
		presetID = 22
		switch direction {
		case model.DirFromLeft:
			filter = "wipe(right)"
		case model.DirFromRight:
			filter = "wipe(left)"
		case model.DirFromTop:
			filter = "wipe(down)"
		case model.DirFromBottom:
			filter = "wipe(up)"
		default:
			filter = "wipe(up)"
		}
	case model.AnimAppear:
		filter = "appear"
		presetID = 1
	default:
		filter = "fade"
		presetID = 10
	}
	return
}
