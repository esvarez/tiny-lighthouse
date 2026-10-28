package sim

import "math"

// Vec is a 2D vector in screen space (y grows downwards).
type Vec struct{ X, Y float64 }

func (a Vec) Add(b Vec) Vec           { return Vec{a.X + b.X, a.Y + b.Y} }
func (a Vec) Sub(b Vec) Vec           { return Vec{a.X - b.X, a.Y - b.Y} }
func (a Vec) Scale(s float64) Vec     { return Vec{a.X * s, a.Y * s} }
func (a Vec) Dot(b Vec) float64       { return a.X*b.X + a.Y*b.Y }
func (a Vec) Cross(b Vec) float64     { return a.X*b.Y - a.Y*b.X }
func (a Vec) Len() float64            { return math.Hypot(a.X, a.Y) }
func (a Vec) Dist(b Vec) float64      { return a.Sub(b).Len() }
func (a Vec) Angle() float64          { return math.Atan2(a.Y, a.X) }
func FromAngle(theta float64) Vec     { return Vec{math.Cos(theta), math.Sin(theta)} }
func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }

// angleDiff returns the signed shortest rotation from a to b, in (-pi, pi].
func angleDiff(a, b float64) float64 {
	d := math.Mod(b-a, 2*math.Pi)
	if d > math.Pi {
		d -= 2 * math.Pi
	} else if d <= -math.Pi {
		d += 2 * math.Pi
	}
	return d
}
