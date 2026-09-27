// Package bm25 is the Okapi BM25 term-score formulas: the per-term document
// score and the query-side term-frequency saturation. A document's score is
// the sum over query terms of Score times QueryTermWeight.
package bm25

import "math"

// Default parameters: K1 and B for the document side, K3 for the query side.
const (
	K1 = 1.2
	B  = 0.75
	K3 = 8
)

// Score is one query term's BM25 score in one document: tf occurrences in a
// document of length dl, a term that df of n documents contain, and an
// average document length of avgdl. The idf is ln(1 + (n-df+0.5)/(df+0.5)),
// which stays positive for terms in more than half the documents.
func Score(tf, df, n, dl, avgdl, k1, b float64) float64 {
	idf := math.Log(1 + (n-df+0.5)/(df+0.5))
	return idf * (tf * (k1 + 1)) / (tf + k1*(1-b+b*dl/avgdl))
}

// QueryTermWeight applies BM25's query-term-frequency saturation to a term
// that occurs qtf times in the query. k3 == 0 uses K3.
func QueryTermWeight(qtf, k3 float64) float64 {
	if k3 == 0 {
		k3 = K3
	}
	return qtf * (k3 + 1) / (qtf + k3)
}
