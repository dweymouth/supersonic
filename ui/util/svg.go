package util

// Contents of this file come from the fyne internal SVG package
// Once https://github.com/fyne-io/fyne/pull/5345 is available in main,
// this file can be retired, and the ColorizeSVG func can be replaced with
// `canvas.ColorizeSVG`

import (
	"bytes"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"image/color"
	"io"
	"strconv"
)

// ColorizeSVG creates a new SVG from a given one by replacing all fill colors by the given color.
func ColorizeSVG(src []byte, clr color.Color) ([]byte, error) {
	rdr := bytes.NewReader(src)
	s, err := svgFromXML(rdr)
	if err != nil {
		return src, fmt.Errorf("could not load SVG, falling back to static content: %v", err)
	}
	if err := s.replaceFillColor(clr); err != nil {
		return src, fmt.Errorf("could not replace fill color, falling back to static content: %v", err)
	}
	colorized, err := xml.Marshal(s)
	if err != nil {
		return src, fmt.Errorf("could not marshal svg, falling back to static content: %v", err)
	}
	return colorized, nil
}

// svg holds the unmarshaled XML from a Scalable Vector Graphic
type svg struct {
	XMLName  xml.Name      `xml:"svg"`
	XMLNS    string        `xml:"xmlns,attr"`
	Width    string        `xml:"width,attr,omitempty"`
	Height   string        `xml:"height,attr,omitempty"`
	ViewBox  string        `xml:"viewBox,attr,omitempty"`
	Paths    []*pathObj    `xml:"path"`
	Rects    []*rectObj    `xml:"rect"`
	Circles  []*circleObj  `xml:"circle"`
	Ellipses []*ellipseObj `xml:"ellipse"`
	Polygons []*polygonObj `xml:"polygon"`
	Groups   []*objGroup   `xml:"g"`
}

type pathObj struct {
	XMLName         xml.Name `xml:"path"`
	Fill            string   `xml:"fill,attr,omitempty"`
	FillOpacity     string   `xml:"fill-opacity,attr,omitempty"`
	Stroke          string   `xml:"stroke,attr,omitempty"`
	StrokeWidth     string   `xml:"stroke-width,attr,omitempty"`
	StrokeLineCap   string   `xml:"stroke-linecap,attr,omitempty"`
	StrokeLineJoin  string   `xml:"stroke-linejoin,attr,omitempty"`
	StrokeDashArray string   `xml:"stroke-dasharray,attr,omitempty"`
	D               string   `xml:"d,attr"`
	Transform       string   `xml:"transform,attr,omitempty"`
}

type rectObj struct {
	XMLName         xml.Name `xml:"rect"`
	Fill            string   `xml:"fill,attr,omitempty"`
	FillOpacity     string   `xml:"fill-opacity,attr,omitempty"`
	Stroke          string   `xml:"stroke,attr,omitempty"`
	StrokeWidth     string   `xml:"stroke-width,attr,omitempty"`
	StrokeLineCap   string   `xml:"stroke-linecap,attr,omitempty"`
	StrokeLineJoin  string   `xml:"stroke-linejoin,attr,omitempty"`
	StrokeDashArray string   `xml:"stroke-dasharray,attr,omitempty"`
	X               string   `xml:"x,attr,omitempty"`
	Y               string   `xml:"y,attr,omitempty"`
	Width           string   `xml:"width,attr,omitempty"`
	Height          string   `xml:"height,attr,omitempty"`
	Transform       string   `xml:"transform,attr,omitempty"`
}

type circleObj struct {
	XMLName         xml.Name `xml:"circle"`
	Fill            string   `xml:"fill,attr,omitempty"`
	FillOpacity     string   `xml:"fill-opacity,attr,omitempty"`
	Stroke          string   `xml:"stroke,attr,omitempty"`
	StrokeWidth     string   `xml:"stroke-width,attr,omitempty"`
	StrokeLineCap   string   `xml:"stroke-linecap,attr,omitempty"`
	StrokeLineJoin  string   `xml:"stroke-linejoin,attr,omitempty"`
	StrokeDashArray string   `xml:"stroke-dasharray,attr,omitempty"`
	CX              string   `xml:"cx,attr,omitempty"`
	CY              string   `xml:"cy,attr,omitempty"`
	R               string   `xml:"r,attr,omitempty"`
	Transform       string   `xml:"transform,attr,omitempty"`
}

type ellipseObj struct {
	XMLName         xml.Name `xml:"ellipse"`
	Fill            string   `xml:"fill,attr,omitempty"`
	FillOpacity     string   `xml:"fill-opacity,attr,omitempty"`
	Stroke          string   `xml:"stroke,attr,omitempty"`
	StrokeWidth     string   `xml:"stroke-width,attr,omitempty"`
	StrokeLineCap   string   `xml:"stroke-linecap,attr,omitempty"`
	StrokeLineJoin  string   `xml:"stroke-linejoin,attr,omitempty"`
	StrokeDashArray string   `xml:"stroke-dasharray,attr,omitempty"`
	CX              string   `xml:"cx,attr,omitempty"`
	CY              string   `xml:"cy,attr,omitempty"`
	RX              string   `xml:"rx,attr,omitempty"`
	RY              string   `xml:"ry,attr,omitempty"`
	Transform       string   `xml:"transform,attr,omitempty"`
}

type polygonObj struct {
	XMLName         xml.Name `xml:"polygon"`
	Fill            string   `xml:"fill,attr,omitempty"`
	FillOpacity     string   `xml:"fill-opacity,attr,omitempty"`
	Stroke          string   `xml:"stroke,attr,omitempty"`
	StrokeWidth     string   `xml:"stroke-width,attr,omitempty"`
	StrokeLineCap   string   `xml:"stroke-linecap,attr,omitempty"`
	StrokeLineJoin  string   `xml:"stroke-linejoin,attr,omitempty"`
	StrokeDashArray string   `xml:"stroke-dasharray,attr,omitempty"`
	Points          string   `xml:"points,attr"`
	Transform       string   `xml:"transform,attr,omitempty"`
}

type objGroup struct {
	XMLName         xml.Name      `xml:"g"`
	ID              string        `xml:"id,attr,omitempty"`
	Fill            string        `xml:"fill,attr,omitempty"`
	Stroke          string        `xml:"stroke,attr,omitempty"`
	StrokeWidth     string        `xml:"stroke-width,attr,omitempty"`
	StrokeLineCap   string        `xml:"stroke-linecap,attr,omitempty"`
	StrokeLineJoin  string        `xml:"stroke-linejoin,attr,omitempty"`
	StrokeDashArray string        `xml:"stroke-dasharray,attr,omitempty"`
	Transform       string        `xml:"transform,attr,omitempty"`
	Paths           []*pathObj    `xml:"path"`
	Circles         []*circleObj  `xml:"circle"`
	Ellipses        []*ellipseObj `xml:"ellipse"`
	Rects           []*rectObj    `xml:"rect"`
	Polygons        []*polygonObj `xml:"polygon"`
	Groups          []*objGroup   `xml:"g"`
}

func replacePathsFill(paths []*pathObj, hexColor string, opacity string) {
	for _, path := range paths {
		if path.Fill != "none" {
			path.Fill = hexColor
			path.FillOpacity = opacity
		}
	}
}

func replaceRectsFill(rects []*rectObj, hexColor string, opacity string) {
	for _, rect := range rects {
		if rect.Fill != "none" {
			rect.Fill = hexColor
			rect.FillOpacity = opacity
		}
	}
}

func replaceCirclesFill(circles []*circleObj, hexColor string, opacity string) {
	for _, circle := range circles {
		if circle.Fill != "none" {
			circle.Fill = hexColor
			circle.FillOpacity = opacity
		}
	}
}

func replaceEllipsesFill(ellipses []*ellipseObj, hexColor string, opacity string) {
	for _, ellipse := range ellipses {
		if ellipse.Fill != "none" {
			ellipse.Fill = hexColor
			ellipse.FillOpacity = opacity
		}
	}
}

func replacePolygonsFill(polys []*polygonObj, hexColor string, opacity string) {
	for _, poly := range polys {
		if poly.Fill != "none" {
			poly.Fill = hexColor
			poly.FillOpacity = opacity
		}
	}
}

func replaceGroupObjectFill(groups []*objGroup, hexColor string, opacity string) {
	for _, grp := range groups {
		replaceCirclesFill(grp.Circles, hexColor, opacity)
		replaceEllipsesFill(grp.Ellipses, hexColor, opacity)
		replacePathsFill(grp.Paths, hexColor, opacity)
		replaceRectsFill(grp.Rects, hexColor, opacity)
		replacePolygonsFill(grp.Polygons, hexColor, opacity)
		replaceGroupObjectFill(grp.Groups, hexColor, opacity)
	}
}

// replaceFillColor alters an svg objects fill color.  Note that if an svg with multiple fill
// colors is being operated upon, all fills will be converted to a single color.  Mostly used
// to recolor Icons to match the theme's IconColor.
func (s *svg) replaceFillColor(color color.Color) error {
	hexColor, opacity := colorToHexAndOpacity(color)
	replacePathsFill(s.Paths, hexColor, opacity)
	replaceRectsFill(s.Rects, hexColor, opacity)
	replaceCirclesFill(s.Circles, hexColor, opacity)
	replaceEllipsesFill(s.Ellipses, hexColor, opacity)
	replacePolygonsFill(s.Polygons, hexColor, opacity)
	replaceGroupObjectFill(s.Groups, hexColor, opacity)
	return nil
}

func svgFromXML(reader io.Reader) (*svg, error) {
	var s svg
	bSlice, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	if err := xml.Unmarshal(bSlice, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func colorToHexAndOpacity(color color.Color) (hexStr, aStr string) {
	r, g, b, a := toNRGBA(color)
	cBytes := []byte{byte(r), byte(g), byte(b)}
	hexStr, aStr = "#"+hex.EncodeToString(cBytes), strconv.FormatFloat(float64(a)/0xff, 'f', 6, 64)
	return hexStr, aStr
}

// toNRGBA converts a color to RGBA values which are not premultiplied, unlike color.RGBA().
func toNRGBA(c color.Color) (r, g, b, a int) {
	// We use UnmultiplyAlpha with RGBA, RGBA64, and unrecognized implementations of Color.
	// It works for all Colors whose RGBA() method is implemented according to spec, but is only necessary for those.
	// Only RGBA and RGBA64 have components which are already premultiplied.
	switch col := c.(type) {
	// NRGBA and NRGBA64 are not premultiplied
	case color.NRGBA:
		r = int(col.R)
		g = int(col.G)
		b = int(col.B)
		a = int(col.A)
	case *color.NRGBA:
		r = int(col.R)
		g = int(col.G)
		b = int(col.B)
		a = int(col.A)
	case color.NRGBA64:
		r = int(col.R) >> 8
		g = int(col.G) >> 8
		b = int(col.B) >> 8
		a = int(col.A) >> 8
	case *color.NRGBA64:
		r = int(col.R) >> 8
		g = int(col.G) >> 8
		b = int(col.B) >> 8
		a = int(col.A) >> 8
	// Gray and Gray16 have no alpha component
	case *color.Gray:
		r = int(col.Y)
		g = int(col.Y)
		b = int(col.Y)
		a = 0xff
	case color.Gray:
		r = int(col.Y)
		g = int(col.Y)
		b = int(col.Y)
		a = 0xff
	case *color.Gray16:
		r = int(col.Y) >> 8
		g = int(col.Y) >> 8
		b = int(col.Y) >> 8
		a = 0xff
	case color.Gray16:
		r = int(col.Y) >> 8
		g = int(col.Y) >> 8
		b = int(col.Y) >> 8
		a = 0xff
	// Alpha and Alpha16 contain only an alpha component.
	case color.Alpha:
		r = 0xff
		g = 0xff
		b = 0xff
		a = int(col.A)
	case *color.Alpha:
		r = 0xff
		g = 0xff
		b = 0xff
		a = int(col.A)
	case color.Alpha16:
		r = 0xff
		g = 0xff
		b = 0xff
		a = int(col.A) >> 8
	case *color.Alpha16:
		r = 0xff
		g = 0xff
		b = 0xff
		a = int(col.A) >> 8
	default: // RGBA, RGBA64, and unknown implementations of Color
		r, g, b, a = unmultiplyAlpha(c)
	}
	return r, g, b, a
}

// unmultiplyAlpha returns a color's RGBA components as 8-bit integers by calling c.RGBA() and then removing the alpha premultiplication.
// It is only used by ToRGBA.
func unmultiplyAlpha(c color.Color) (r, g, b, a int) {
	red, green, blue, alpha := c.RGBA()
	if alpha != 0 && alpha != 0xffff {
		red = (red * 0xffff) / alpha
		green = (green * 0xffff) / alpha
		blue = (blue * 0xffff) / alpha
	}
	// Convert from range 0-65535 to range 0-255
	r = int(red >> 8)
	g = int(green >> 8)
	b = int(blue >> 8)
	a = int(alpha >> 8)
	return r, g, b, a
}
