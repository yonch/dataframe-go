// Copyright 2018-20 PJ Engineering and Business Solutions Pty. Ltd. All rights reserved.

package pandas

import (
	"context"
	"math"

	dataframe "github.com/rocketlaunchr/dataframe-go"
)

func interpolateSeriesFloat64(ctx context.Context, fs *dataframe.SeriesFloat64, opts InterpolateOptions) (*dataframe.OrderedMapIntFloat64, error) {

	if !opts.DontLock {
		fs.Lock()
		defer fs.Unlock()
	}

	var omap *dataframe.OrderedMapIntFloat64
	if !opts.InPlace {
		omap = dataframe.NewOrderedMapIntFloat64()
	}

	var (
		limArea *InterpolationLimitArea
		r       *dataframe.Range
	)

	if limArea != nil {
		limArea = opts.LimitArea
	}

	r = &dataframe.Range{}
	if opts.R != nil {
		r = opts.R
	}

	start, end, err := r.Limits(len(fs.Values))
	if err != nil {
		return nil, err
	}

	startOfSeg := start

	// Step 1: Find ranges that are nil values in between

	for {

		if startOfSeg >= end-1 {
			break
		}

		if err := ctx.Err(); err != nil {
			return nil, err
		}

		var (
			left  *int
			right *int
		)

		for i := startOfSeg; i <= end; i++ {
			currentVal := fs.Values[i]
			if !math.IsNaN(currentVal) {
				if left == nil {
					left = &[]int{i}[0]
				} else {
					right = &[]int{i}[0]
					break
				}
			}
		}

		if left != nil && right != nil {
			if opts.LimitArea == nil || opts.LimitArea.has(Inner) {
				// Fill Inner range

				switch opts.Method {
				case ForwardFill:
					fillFn := func(row int) float64 {
						return fs.Values[*left]
					}
					err := fill(ctx, fillFn, fs, omap, *left, *right, opts.LimitDirection, opts.Limit)
					if err != nil {
						return nil, err
					}
				case BackwardFill:
					fillFn := func(row int) float64 {
						return fs.Values[*right]
					}
					err := fill(ctx, fillFn, fs, omap, *left, *right, opts.LimitDirection, opts.Limit)
					if err != nil {
						return nil, err
					}
				case Linear:
					grad := (fs.Values[*right] - fs.Values[*left]) / float64(*right-*left)
					c := fs.Values[*left] + grad
					fillFn := func(row int) float64 {
						return grad*float64(row) + c
					}
					err := fill(ctx, fillFn, fs, omap, *left, *right, opts.LimitDirection, opts.Limit)
					if err != nil {
						return nil, err
					}
				}

			}
			startOfSeg = *right
		} else {
			// Outer
			break
		}

	}

	if opts.InPlace {
		return nil, nil
	} else {
		return omap, nil
	}
}
