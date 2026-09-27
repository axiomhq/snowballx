// Package greek is the Snowball Greek stemmer as generated Go code on the
// github.com/blevesearch/snowballstem runtime, the runtime that stems every
// other Snowball language in Go. blevesearch/snowballstem ships no greek
// package, so this one is generated the same way its packages are:
//
//	git clone https://github.com/snowballstem/snowball && cd snowball
//	git checkout v3.1.1 # cd195b51e948a902a4312f023f4a14392516a543
//	make snowball
//	./snowball algorithms/greek.sbl -go -P greek \
//	    -gor github.com/blevesearch/snowballstem -o greek_stemmer
//
// greek_stemmer.go is that output, unedited; regenerate rather than edit
// it. On snowball-data's greek/voc.txt (90,727 words) it matches
// greek/output.txt exactly; stem_test.go keeps a sample of those pairs.
package greek
