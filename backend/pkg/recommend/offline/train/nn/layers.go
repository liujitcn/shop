package nn

import (
	"github.com/chewxy/math32"
)

type Layer interface {
	Parameters() []*Tensor
	Forward(x *Tensor) *Tensor
	SetJobs(jobs int)
}

type Model Layer

type LinearLayer struct {
	W    *Tensor
	B    *Tensor
	jobs int
}

func NewLinear(in, out int) Layer {
	bound := 1.0 / math32.Sqrt(float32(in))
	return &LinearLayer{
		W: Uniform(-bound, bound, in, out),
		B: Zeros(out),
	}
}

func (l *LinearLayer) Forward(x *Tensor) *Tensor {
	return Add(MatMul(x, l.W, false, false, l.jobs), l.B)
}

func (l *LinearLayer) Parameters() []*Tensor {
	return []*Tensor{l.W, l.B}
}

func (l *LinearLayer) SetJobs(jobs int) {
	l.jobs = max(1, jobs)
}

type flattenLayer struct{}

func NewFlatten() Layer {
	return &flattenLayer{}
}

func (f *flattenLayer) Parameters() []*Tensor {
	return nil
}

func (f *flattenLayer) Forward(x *Tensor) *Tensor {
	return Flatten(x)
}

func (f *flattenLayer) SetJobs(int) {}

type EmbeddingLayer struct {
	W *Tensor
}

func NewEmbedding(n int, shape ...int) Layer {
	wShape := append([]int{n}, shape...)
	return &EmbeddingLayer{
		W: Normal(0, 0.01, wShape...),
	}
}

func (e *EmbeddingLayer) Parameters() []*Tensor {
	return []*Tensor{e.W}
}

func (e *EmbeddingLayer) Forward(x *Tensor) *Tensor {
	return Embedding(e.W, x)
}

func (e *EmbeddingLayer) SetJobs(int) {}

type sigmoidLayer struct{}

func NewSigmoid() Layer {
	return &sigmoidLayer{}
}

func (s *sigmoidLayer) Parameters() []*Tensor {
	return nil
}

func (s *sigmoidLayer) Forward(x *Tensor) *Tensor {
	return Sigmoid(x)
}

func (s *sigmoidLayer) SetJobs(int) {}

type reluLayer struct{}

func NewReLU() Layer {
	return &reluLayer{}
}

func (r *reluLayer) Parameters() []*Tensor {
	return nil
}

func (r *reluLayer) Forward(x *Tensor) *Tensor {
	return ReLu(x)
}

func (r *reluLayer) SetJobs(int) {}

type Sequential struct {
	Layers []Layer
}

func NewSequential(layers ...Layer) Model {
	return &Sequential{Layers: layers}
}

func (s *Sequential) Parameters() []*Tensor {
	var params []*Tensor
	for _, l := range s.Layers {
		params = append(params, l.Parameters()...)
	}
	return params
}

func (s *Sequential) Forward(x *Tensor) *Tensor {
	for _, l := range s.Layers {
		x = l.Forward(x)
	}
	return x
}

func (s *Sequential) SetJobs(jobs int) {
	for _, l := range s.Layers {
		l.SetJobs(jobs)
	}
}

type Attention struct {
	W    Layer
	H    *Tensor
	jobs int
}

func NewAttention(dimensions, k int) *Attention {
	return &Attention{
		W: NewLinear(dimensions, k),
		H: Normal(0, 0.01, k, dimensions),
	}
}

func (a *Attention) Parameters() []*Tensor {
	var params []*Tensor
	params = append(params, a.H)
	params = append(params, a.W.Parameters()...)
	return params
}

func (a *Attention) Forward(x *Tensor) *Tensor {
	return Mul(
		Softmax(MatMul(ReLu(a.W.Forward(x)), a.H, false, false, a.jobs), 1),
		x,
	)
}

func (a *Attention) SetJobs(jobs int) {
	a.W.SetJobs(jobs)
	a.jobs = max(1, jobs)
}
