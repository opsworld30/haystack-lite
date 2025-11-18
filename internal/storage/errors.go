package storage

import "errors"

var (
	ErrNeedleNotFound = errors.New("needle not found")
	ErrVolumeNotFound = errors.New("volume not found")
	ErrVolumeFull     = errors.New("volume is full")
	ErrCRCMismatch    = errors.New("crc checksum mismatch")
	ErrReadOnly       = errors.New("storage is read-only")
	ErrInvalidNeedle  = errors.New("invalid needle")
)
