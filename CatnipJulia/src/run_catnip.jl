using Colors
using DataStructures
using GLMakie

set_theme!(theme_black())

GLMakie.activate!(;
  vsync=false,
  framerate=60.0,
  float=false,
  pause_renderloop=false,
  focus_on_show=false,
  decorated=true,
  title="catnip"
)

function run_catnip(; timeout=false)
  println("Start")
  # 6 seconds at 60 samples per second out of catnip
  numSets = 300
  # numSets = 120

  fig = Figure(fontsize=14; size=(740, 976))
  ax1 = Axis3(fig[1, 1]; aspect=(1.5, 1.00, 0.25), elevation=(2 * pi) / 4, perspectiveness=0.25, azimuth=0 * pi)

  hidedecorations!(ax1)
  hidespines!(ax1)

  println("Allocate")

  data = DataStructures.CircularBuffer{Vector{Float64}}(numSets)

  push!(data, [0.0, 0.0])
  push!(data, [0.0, 0.0])

  z = Observable(mapreduce(permutedims, vcat, data))

  mSize = @lift(Vec3f.(1, 1, $z[:]))

  zZ = @lift(0 * $z)

  sets = @lift(size($z, 1))
  bars = @lift(size($z, 2))

  x = @lift(range(1, $sets, $sets))
  y = @lift(range(1, $bars, $bars))

  idx = Observable(1)

  function makeColorMap(s, b, x)
    mat = zeros(s, b)

    for i = 0:s-1
      # v = 1 - min(0.8, i / s)
      v = 1.0 - (i / s)
      # v = i / s

      for j = 0:b-1
        mat[i+(j*s)+1] = v
      end
    end

    for j = 0:b-1
      mat[x+(j*s)] = 0.0
    end

    return mat
  end

  function makeColorMap2(data, sets, bars, idx)
    d = data[:]

    for n in eachindex(d)
      d[n] > 100.0 && (d[n] = 100.0)
      # d[n] < 1.0 && (d[n] = 100.0)
    end

    # for j = 0:bars-1
    #   d[idx+(j*sets)] = 100.0
    # end

    for j = 0:bars-1
      d[1+(j*sets)] = 100.0
      # d[idx+(j*sets)] = 100.0
    end

    d[idx] = 0.0
    d[idx+((bars-1)*sets)] = 100.0

    return d
  end

  zC = @lift($z[:])

  staticColorMap = @lift(makeColorMap2($z, $sets, $bars, $idx))


  function updateZ()
    z[] = mapreduce(permutedims, vcat, data)
  end

  rectMesh = Rect3f(Vec3f(-0.5, -0.5, 0), Vec3f(1, 1, 1))

  display(fig)

  mymagma = GLMakie.to_colormap(:magma)
  # mymagma = GLMakie.to_colormap(:BuPu_9)
  mymagma[1] = RGBA(0.0, 0.0, 0.0, 0.0)

  command = `catnip -d spotify -r 122880 -n 2048 -sm 4 -sas 6 -sf 45 -i -raw -rawb 82 -rawm`
  # command = `catnip -d "Google Chrome" -r 122880 -n 2048 -sas 6 -sf 45 -i -raw -rawb 60 -rawm`
  #command = `go run ./cmd/catnip -d spotify -r 122880 -n 2048 -sas 5 -sf 45 -i -raw -rawb 50 -rawm`


  try
    println("Open StdOut")
    open(command, "r", stdout) do io
      count = 0
      println("Loop")
      while count < numSets + 2
        count += 1

        line = readline(io)
        line = strip(line)
        elms = split(line, " ")
        elms = [elm for elm in elms if !isempty(elm)]
        nums = map(x -> parse(Float64, strip(x)), elms)
        #println(nums)
        pushfirst!(data, reverse(nums))
      end

      updateZ()

      limits!(ax1, 0, sets[], 0, bars[], 0, 100)

      meshscatter!(ax1, x, y, zZ; marker=rectMesh, color=staticColorMap,
        markersize=mSize, colormap=mymagma,
        # markersize=mSize, colormap=:plasma,
        shading=MultiLightShading)

      idx[] = numSets


      #while !eof(io) && (count < numSets + 4 || !timeout)
      while !eof(io) && !timeout
        count += 1
        if count >= numSets
          count = 0
        end

        line = readline(io)
        line = strip(line)
        elms = split(line, " ")
        elms = [elm for elm in elms if !isempty(elm)]
        nums = map(x -> parse(Float64, strip(x)), elms)
        #nums = reverse(nums)

        # idx[] = numSets - count
        #println(nums)

        # data[idx[]] = nums

        pushfirst!(data, nums)

        updateZ()
      end
    end
  catch e
    @show e
  end
end

run_catnip()
