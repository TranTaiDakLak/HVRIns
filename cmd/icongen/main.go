package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"os"

	"golang.org/x/image/draw"
)

// Tạo file ICO từ PNG, chứa nhiều kích thước cho Windows taskbar
func main() {
	srcPath := "build/appicon.png"
	dstPath := "build/windows/icon.ico"

	// Đọc PNG gốc
	f, err := os.Open(srcPath)
	if err != nil {
		fmt.Println("Lỗi mở file:", err)
		os.Exit(1)
	}
	defer f.Close()

	src, err := png.Decode(f)
	if err != nil {
		fmt.Println("Lỗi decode PNG:", err)
		os.Exit(1)
	}

	// Các kích thước cần cho Windows ICO
	sizes := []int{256, 128, 64, 48, 32, 16}

	// Tạo PNG data cho từng kích thước
	pngDataList := make([][]byte, len(sizes))
	for i, size := range sizes {
		resized := resizeImage(src, size)
		var buf bytes.Buffer
		if err := png.Encode(&buf, resized); err != nil {
			fmt.Printf("Lỗi encode PNG %dx%d: %v\n", size, size, err)
			os.Exit(1)
		}
		pngDataList[i] = buf.Bytes()
	}

	// Ghi ICO file
	ico := buildICO(sizes, pngDataList)
	if err := os.WriteFile(dstPath, ico, 0644); err != nil {
		fmt.Println("Lỗi ghi ICO:", err)
		os.Exit(1)
	}

	fmt.Printf("Đã tạo %s (%d bytes, %d kích thước)\n", dstPath, len(ico), len(sizes))
}

// Resize ảnh về kích thước target
func resizeImage(src image.Image, size int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

// Build ICO file format
func buildICO(sizes []int, pngData [][]byte) []byte {
	var buf bytes.Buffer
	n := len(sizes)

	// ICO Header (6 bytes)
	binary.Write(&buf, binary.LittleEndian, uint16(0))     // reserved
	binary.Write(&buf, binary.LittleEndian, uint16(1))     // type: ICO
	binary.Write(&buf, binary.LittleEndian, uint16(n))     // count

	// Tính offset cho data (header 6 bytes + entries n*16 bytes)
	dataOffset := 6 + n*16

	// ICO Directory entries (16 bytes mỗi entry)
	for i, size := range sizes {
		w := byte(size)
		h := byte(size)
		if size == 256 {
			w = 0 // 0 = 256 trong ICO format
			h = 0
		}
		buf.WriteByte(w)                                              // width
		buf.WriteByte(h)                                              // height
		buf.WriteByte(0)                                              // color palette
		buf.WriteByte(0)                                              // reserved
		binary.Write(&buf, binary.LittleEndian, uint16(1))            // color planes
		binary.Write(&buf, binary.LittleEndian, uint16(32))           // bits per pixel
		binary.Write(&buf, binary.LittleEndian, uint32(len(pngData[i]))) // data size
		binary.Write(&buf, binary.LittleEndian, uint32(dataOffset))   // data offset

		dataOffset += len(pngData[i])
	}

	// PNG data cho từng kích thước
	for _, data := range pngData {
		buf.Write(data)
	}

	return buf.Bytes()
}
