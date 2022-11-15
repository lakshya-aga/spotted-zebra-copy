package mc

import (
	"math"
	"time"

	"golang.org/x/exp/rand"
	"gonum.org/v1/gonum/stat/distuv"
)

// Define HypHyp model.
type HypHyp struct {
	Sigma, Alpha, Beta, Kappa, Rho float64
}

// Simulate a HypHyp model price path for a given vector of timesteps and stock price normal variates.
// The normal variates are used when it is required to generated correlated price paths of two or more assets.
// For a single price path z1 should be nil.
func (m HypHyp) Path(dt, z1 []float64) []float64 {
	var f, g, y, u, x float64
	N := len(dt)
	// Initialise price path
	r := make([]float64, N+1)
	r[0] = 0.0
	// Pre compute some repeated constants used in SDE
	a := 0.5 * m.Sigma * m.Sigma
	b1 := m.Beta
	b2 := b1 * b1
	// Initialise Std Normal generator
	d := distuv.Normal{Mu: 0.0, Sigma: 1.0, Src: rand.NewSource(uint64(time.Now().UnixNano()))}
	// We need two correlated std normal variates for path simulation. If the normal variate for the stock price SDE is not given, generate it here.
	if z1 == nil {
		z1 = make([]float64, N)
		for i := 0; i < N; i++ {
			z1[i] = d.Rand()
		}
	}
	// Generate the state variable SDE normal variate
	z2 := make([]float64, N)
	for i := range z2 {
		z2[i] = m.Rho*z1[i] + math.Sqrt(1.0-m.Rho*m.Rho)*d.Rand()
	}

	// Generate Euler-Mauryama path for log price
	for i := 0; i < N; i++ {
		x = math.Exp(r[i])
		f = ((1.0-b1+b2)*x + (b1-1)*(math.Sqrt(x*x+b2*(1.0-x)*(1.0-x))-b1)) / b1
		g = y + math.Sqrt(y*y+1.0)
		u = f * g / x
		r[i+1] = r[i] - a*dt[i]*u*u + u*math.Sqrt(dt[i])*z1[i]
		y = y*math.Exp(-m.Kappa*dt[i]) + m.Alpha*math.Sqrt(1.0-math.Exp(-2.0*m.Kappa*dt[i]))*z2[i] //*math.Sqrt(dt[i])
	}
	// Convert log price to price
	for i, v := range r {
		r[i] = math.Exp(v)
	}
	return r
}

// // String method to satisfy Stringer interface for printing purposes
// func (m HypHyp) String() string {
// 	return fmt.Sprintf("Sigma: %v\nAlpha: %v\nBeta: %v\nKappa: %v\nRho: %v\n", m.Sigma, m.Alpha, m.Beta, m.Kappa, m.Rho)
// }

// Constructor for HypHyp model
func NewHypHyp() HypHyp {
	return HypHyp{Sigma: 0.40, Alpha: 0.01, Beta: 0.01, Rho: 0.0, Kappa: 5.0}
}

// Get transformed parameters. Return parameters transformed to the domain (-Inf, Inf).
func (m HypHyp) Get() []float64 {
	p := make([]float64, 5)
	p[0], p[1], p[2], p[3] = math.Log(m.Sigma), math.Log(m.Alpha), math.Log(m.Beta), math.Log(m.Kappa)
	p[4] = math.Atanh(m.Rho)
	return p
}

// Create a model for the given transformed parameters
func (m HypHyp) Set(p []float64) Model {
	m.Sigma, m.Alpha, m.Beta, m.Kappa = math.Exp(p[0]), math.Exp(p[1]), math.Exp(p[2]), math.Exp(p[4])
	m.Rho = math.Tanh(p[4])
	return m
}

// Compute model implied volatility
func (m HypHyp) IVol(k, T float64) float64 {
	a := m.Alpha * m.Kappa * T
	h := math.Sqrt(1.0+a) - math.Sqrt(a)
	v_watanabe := m.watanabe(k, T)
	v_watanabe_ATM := m.watanabe(1.0, T)
	v_fouque_ATM := m.fouqueATM(T)
	ivol := v_watanabe * ((1.0-h)*v_fouque_ATM/v_watanabe_ATM + h)
	return ivol
}

// Helper function for implied volatility calculation.
func (m HypHyp) fouqueATM(T float64) float64 {
	u := m.Kappa * T
	a := m.Alpha * m.Alpha
	s := math.Sqrt((math.Exp(-2.0*u)-1.0)*a/u + 2.0*a + 1.0)
	v := m.Sigma*s - (m.Alpha*(a*a-7.0*a-1.0)*m.Rho*m.Sigma*m.Sigma)/(s*math.Sqrt(2.0*m.Kappa))
	return v
}

// Helper function for implied volatility calculation.
func (m HypHyp) watanabe(k, T float64) float64 {
	a, b, s, r, h := m.Alpha, m.Beta, m.Sigma, m.Rho, m.Kappa
	a2, r2 := a*a, r*r
	h1 := math.Pow(h, 1.5)
	h2 := h * h
	u0 := h * T
	u02 := u0 * u0
	T2 := T * T
	T3 := T2 * T
	s2 := s * s
	u, u1 := math.Exp(-u0), math.Exp(u0)
	uu := u * u
	u2 := u1 * u1
	st := math.Sqrt(T)
	b1 := b * (b - 1.0)
	z := (k - 1.0) / (s * st)
	z2 := z * z
	f1, f2, f3, f4 := b, b1, -3.0*b1, -3.0*b1*(b*b-4.0)
	f12 := f1 * f1
	f13 := f12 * f1
	f22 := f2 * f2
	f44 := f4 * f4 * f4 * f4

	S1 := (z * s) / (2.0 * st) * ((f1-1.0)*s*T + math.Sqrt(8.0)*a*r*(u0+u-1)/(h1*T))

	s21 := 12.0 * math.Sqrt(2.0) * u1 * f1 * a * h1 * r * s * T2 * (u1*(u0-1.0) + 1.0)
	s22 := -u0 * (u2*(f12-2.0*f2-1.0)*T3*h2*s2 - 6.0*a2*r2*(2*u2*u02-5.0*u2*u0+u0-8.0*u1+6.0*u2+2.0))
	s23 := (-6.0 * a2) * (2.0*u2*u02*u0*(r2-1) + u02*(-9.0*u2*r2+r2+5.0*u2-1.0) - 2.0*u0*(u1-1.0)*(-7.0*u1*r2+r2+3.0*u1-1.0) - 4.0*(u1-1.0)*(u1-1.0)*r2)
	s24 := z2 * (-12.0*math.Sqrt(2.0)*u1*a*h1*r*s*T2*(u1*(u0-1.0)+1.0) - u0*(u2*u02*T*s2*(2.0*f12+6.0*f1-4.0*f2-8.0)-6.0*a2*r2*(4.0*u2*u0+8.0*u1-6.0*u2-2.0)) - 6.0*a2*(u02*(12*u2*r2-4.0*u2)+8.0*(u1-1.0)*(u1-1.0)*r2-2.0*(u1-1.0)*u0*(11.0*u1*r2-r2-3.0*u1+1.0)))

	S2 := (s * uu) / (24.0 * u02 * u0) * (s21 + s22 + s23 + s24)

	S3 := (math.Pow(T, 1.5) * z * s2 * s2) / 48.0 * (-f13 + f12 + (2.0*f2+3.0)*f1 - 2.0*f2 + 2.0*f3 - 3.0 + 2.0*z2*(f13+f12+(4.0-2.0*f2)*f1-2.0*f2+f3-6.0))

	S41 := 8.0 * z2 * z2 * (19.0*f12*f12 + 15.0*f13 + (20.0-46.0*f2)*f12 + 6.0*(3.0*f3-5.0*f2+15.0)*f1 - 40.0*f2 + 16.0*f22 + 15.0*f3 - 6.0*f4 - 144.0)
	S42 := -2.0 * z2 * (11.0*f44 + 30.0*f13 + (20.0-44.0*f2)*f12 + 6.0*(12.0*f3-10.0*f2-45.0)*f1 + 140.0*f2 + 44.0*f22 - 60.0*f3 + 36.0*f4 + 209.0)
	S43 := -3.0 * (3.0*f12*f12 - 2.0*(6.0*f2+5.0)*f12 + 16.0*f3*f1 + 12.0*f22 + 20.0*f2 + 8.0*f4 + 7.0)
	S4 := (-T2 * s2 * s2 * s) / 5760.0 * (S41 + S42 + S43)

	out := s + S1 + S2 + S3 + S4
	return out
}
