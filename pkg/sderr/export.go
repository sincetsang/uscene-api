package sderr

import (
	stderrors "errors"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

var (
	// New
	New      = errors.New
	Newf     = errors.Errorf
	Sentinel = stderrors.New

	// With
	WithStack    = errors.WithStack
	WithMessage  = errors.WithMessage
	WithMessagef = errors.WithMessagef
	WithPrefix   = multierror.Prefix
	Wrap         = errors.Wrap
	Wrapf        = errors.Wrapf

	// Cause
	Cause  = errors.Cause
	Unwrap = errors.Unwrap

	// Is/As
	Is = errors.Is
	As = errors.As

	// Multiple
	Append     = multierror.Append
	Flatten    = multierror.Flatten
	ListFormat = multierror.ListFormatFunc
)

type (
	Frame           = errors.Frame
	StackTrace      = errors.StackTrace
	MultipleError   = multierror.Error
	ErrorFormatFunc = multierror.ErrorFormatFunc
	Group           = multierror.Group
)
