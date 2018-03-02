package qrpie

import (
	"math"
	"sort"
)

type cluster struct {
	heaps [][]point
}

func NewCluster() *cluster {
	return &cluster{
		heaps: make([][]point, 0, 10),
	}
}

func (c *cluster) add(p point) {
	dis := math.MaxFloat64
	for i, ps := range c.heaps {
		for _, p1 := range ps {
			if distant(p1, p) <= 2 {
				c.heaps[i] = append(c.heaps[i], p)
				return
			}
			if distant(p1, p) < dis {
				dis = distant(p1, p)
			}

		}
	}
	c.heaps = append(c.heaps, []point{p})

}

func (c *cluster) Less(i, j int) bool {
	if len(c.heaps[i]) < len(c.heaps[j]) {
		return true
	} else {
		return false
	}
}

func (c *cluster) Len() int {
	return len(c.heaps)
}

func (c *cluster) Swap(i, j int) {
	buf := c.heaps[i]
	c.heaps[i] = c.heaps[j]
	c.heaps[j] = buf
}

func (c *cluster) getMaxThreeHeapCenter() []point {
	if len(c.heaps) < 3 {
		return nil
	}
	centers := make([]point, 3, 3)
	sort.Sort(c)
	for i := 0; i < 3; i++ {
		index := len(c.heaps) - i - 1
		centers[i] = mean(c.heaps[index])
	}
	return centers
}

func (c *cluster) getLenth() int {
	return len(c.heaps)
}

func (c *cluster) getVariance() float64 {
	if c.getLenth() == 0 {
		return 0
	}
	sum := 0
	for _, ps := range c.heaps {
		sum = sum + len(ps)
	}
	mean := float64(sum) / float64(c.getLenth())
	v := 0.0
	for _, ps := range c.heaps {
		v = v + math.Pow(float64(len(ps))-mean, 2)
	}
	return v / float64(c.getLenth())
}

func mean(points []point) point {
	sum := point{0, 0}
	l := float64(len(points))
	for _, p := range points {
		sum = pointAdd(sum, p)
	}
	return point{sum.x / l, sum.y / l}
}
