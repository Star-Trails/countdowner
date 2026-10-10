//go:build !windows

package desktop

import (
	"errors"

	"countdowner/internal/model"
)

const CornerRadius = 8

type Controller struct {
	OnResize   func(model.Position, model.Size)
	OnPosition func(model.Position)
	OnRedraw   func()
}

func Available() bool { return false }

func (c *Controller) Attach(uintptr, model.Position, int, int) error {
	return errors.New("桌面嵌入仅支持 Windows")
}
func (c *Controller) Ensure() error                { return errors.New("桌面嵌入仅支持 Windows") }
func (c *Controller) Restore(model.Position) error { return c.Ensure() }
func (c *Controller) SetSize(int, int) error       { return c.Ensure() }
func (c *Controller) VisualScale() float32         { return 1 }
func (c *Controller) Size() model.Size             { return model.Size{} }
