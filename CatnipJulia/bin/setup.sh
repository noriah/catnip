#!/bin/bash
DIR="$(dirname "$0")"
julia --startup-file=no -e "using Pkg; Pkg.activate(\"$DIR/..\"); Pkg.instantiate(); include(\"$DIR/../build_sysimage.jl\")"
