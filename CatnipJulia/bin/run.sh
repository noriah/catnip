#!/bin/bash
DIR="$(dirname "$0")"
julia -J $DIR/../CatnipJuliaSysimagePrecompile.so --startup-file=no -e 'CatnipJulia.run_catnip()'
