package main

import (
	"fmt"
	"image"
	"os"
	"strconv"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/muesli/coral"
	"github.com/nfnt/resize"
)

var (
	imageCmd = &coral.Command{
		Use:   "image <key> <image>",
		Short: "sets an image on a key",
		RunE: func(cmd *coral.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("image requires the key-index and an image")
			}

			key, err := strconv.ParseInt(args[0], 10, 8)
			if err != nil {
				return fmt.Errorf("supplied parameter is not a valid number")
			}

			f, err := os.Open(args[1])
			if err != nil {
				return err
			}
			defer f.Close() //nolint:errcheck // r/o file

			img, _, err := image.Decode(f)
			if err != nil {
				return err
			}

			return d.SetImage(uint8(key), resize.Resize(72, 72, img, resize.Lanczos3))
		},
	}
)

func init() {
	RootCmd.AddCommand(imageCmd)
}
