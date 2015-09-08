waveletmatrix
=============

[![Godoc](https://godoc.org/github.com/hideo55/go-waveletmatrix?status.png)](https://godoc.org/github.com/hideo55/go-waveletmatrix)
[![Build Status](https://travis-ci.org/hideo55/go-waveletmatrix.svg?branch=master)](https://travis-ci.org/hideo55/go-waveletmatrix)
[![Coverage Status](https://coveralls.io/repos/hideo55/go-waveletmatrix/badge.svg?branch=master&service=github)](https://coveralls.io/github/hideo55/go-waveletmatrix?branch=master)

Description
-----------

[The Wavelet Matrix](http://www.dcc.uchile.cl/~gnavarro/ps/spire12.4.pdf) implementation for Go..

Usage
-----

```go
package main

import (
    "fmt"

    "github.com/hideo55/go-waveletmatrix"
)

func main() {
    src := []uint64{1, 3, 1, 4, 2, 1, 10}
    wm, err := waveletmatrix.NewWM(src)
    if err != nil {
        // Failed to build wavelet-matrix
    }
    val, _ := wm.Lookup(3) 
    fmt.Println(val) // 4 ... src[3]
    rank, _ := wm.Rank(2, 6) 
    fmt.Println(rank) // 1 ... The number of 2 in src[0, 6)
    ranklt := wm.RankLessThan(3, 6)
    fmt.Println(ranklt) /// 4 ... The frequency of characters c' < c in src[0, 6)
    rankmt := wm.RankMoreThan(3, 6)
    fmt.Println(rankmt) /// 1 ... The frequency of characters c' > c in the src[0, 6)
    pos, _ := wm.Select(1, 3) // = 5 ... The third 1 appeared in src[5]
    fmt.Println(pos)
}
```

Supported version
-----------------

Go 1.4 or later

License
--------

MIT License
