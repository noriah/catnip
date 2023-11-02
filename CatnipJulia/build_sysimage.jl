using Pkg
Pkg.activate(@__DIR__)
using PackageCompiler

PackageCompiler.create_sysimage(["CatnipJulia"]; sysimage_path="CatnipJuliaSysimagePrecompile.so", precompile_execution_file="precompile_script.jl")
