absorbingmarkovchain
========

[![GoDoc Reference](https://godoc.org/github.com/ebonetti/absorbingmarkovchain?status.svg)](http://godoc.org/github.com/ebonetti/absorbingmarkovchain)
[![Build Status](https://travis-ci.org/ebonetti/absorbingmarkovchain.svg?branch=master)](https://travis-ci.org/ebonetti/absorbingmarkovchain)
[![Coverage Status](https://coveralls.io/repos/ebonetti/absorbingmarkovchain/badge.svg?branch=master)](https://coveralls.io/r/ebonetti/absorbingmarkovchain?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/ebonetti/absorbingmarkovchain)](https://goreportcard.com/report/github.com/ebonetti/absorbingmarkovchain)

Description
-----------

absorbingmarkovchain is a golang package that defines primitives for computing absorption probabilities of absorbing markov chains.

Installation
------------

This package can be installed with the go get command:

    go get github.com/ebonetti/absorbingmarkovchain

Dependencies
-------------

This package depends on `PETSc`. The associated dockerfile provides a complete environment in which use this package, such docker image can be found at [ebonetti/golang-petsc](https://hub.docker.com/r/ebonetti/golang-petsc/). Otherwise `PETSc` can be installed following the same steps as in the dockerfile or in [the PETSc installation page](https://www.mcs.anl.gov/petsc/documentation/installation.html).

Documentation
-------------

API documentation can be found in the [associated godoc reference](https://godoc.org/github.com/ebonetti/absorbingmarkovchain).