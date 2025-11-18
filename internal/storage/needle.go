package storage

import (
	"encoding/binary"
	"hash/crc32"
	"io"
)

const (
	NeedleHeaderSize = 8 + 4 + 4 + 8 + 1 // ID + Cookie + DataSize + CreateTime + Flags
	NeedleFooterSize = 4                 // CRC32
	NeedleMagic      = 0x1234
)

type Needle struct {
	ID         uint64
	Cookie     uint32
	Data       []byte
	DataSize   uint32
	Flags      uint8
	CreateTime int64
	FileName   string // 文件名
	MimeType   string // MIME 类型
	MD5        string // MD5 哈希
}

type NeedleInfo struct {
	Offset   int64
	Size     uint32
	Flags    uint8
	VolumeID uint32
}

func (n *Needle) IsDeleted() bool {
	return n.Flags&0x01 != 0
}

func (n *Needle) SetDeleted() {
	n.Flags |= 0x01
}

func (n *Needle) Write(w io.Writer) error {
	// Header
	if err := binary.Write(w, binary.BigEndian, n.ID); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, n.Cookie); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, n.DataSize); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, n.CreateTime); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, n.Flags); err != nil {
		return err
	}

	// Data
	if _, err := w.Write(n.Data); err != nil {
		return err
	}

	// CRC32
	crc := crc32.ChecksumIEEE(n.Data)
	if err := binary.Write(w, binary.BigEndian, crc); err != nil {
		return err
	}

	return nil
}

func ReadNeedleFrom(r io.Reader) (*Needle, error) {
	n := &Needle{}

	if err := binary.Read(r, binary.BigEndian, &n.ID); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.BigEndian, &n.Cookie); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.BigEndian, &n.DataSize); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.BigEndian, &n.CreateTime); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.BigEndian, &n.Flags); err != nil {
		return nil, err
	}

	n.Data = make([]byte, n.DataSize)
	if _, err := io.ReadFull(r, n.Data); err != nil {
		return nil, err
	}

	var crc uint32
	if err := binary.Read(r, binary.BigEndian, &crc); err != nil {
		return nil, err
	}

	if crc32.ChecksumIEEE(n.Data) != crc {
		return nil, ErrCRCMismatch
	}

	return n, nil
}

func (n *Needle) Size() int64 {
	return int64(NeedleHeaderSize + n.DataSize + NeedleFooterSize)
}
