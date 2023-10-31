using Colors
using DataStructures: CircularBuffer
using GLMakie

GLMakie.activate!()

set_window_config!(;
  vsync=false,
  framerate=60.0,
  float=true,
  pause_renderloop=false,
  focus_on_show=false,
  decorated=true,
  title="Makie"
)

numSets = 120

fig = Figure(resolution=(1200, 800), fontsize=14)
ax1 = Axis3(fig[1, 1]; aspect=(2, 1, 0.25), elevation=pi / 6, perspectiveness=0.5)

data = CircularBuffer{Vector{Float64}}(numSets)

push!(data, [0.0, 0.0])
push!(data, [0.0, 0.0])

z = Observable(mapreduce(permutedims, vcat, data))

mSize = @lift(Vec3f.(1, 1, $z[:]))

zZ = @lift(0 * $z)

sets = @lift(size($z, 1))
bars = @lift(size($z, 2))

x = @lift(range(1, $sets, $sets))
y = @lift(range(1, $bars, $bars))

function makeColorMap(s, b)
  mat = zeros(s, b)

  for j = 0:b-1
    mat[(j*s)+1] = 0
  end

  for i = 1:s-1
    # v = 1 - min(0.8, i / s)
    v = 1 - (i / s)
    # v = i / s

    for j = 0:b-1
      mat[(j*s)+i+1] = v
    end
  end

  return mat
end

zC = @lift($z[:])

staticColorMap = @lift(makeColorMap($sets, $bars)[:])


function updateZ()
  z[] = mapreduce(permutedims, vcat, data)
end

rectMesh = Rect3f(Vec3f(-0.5, -0.5, 0), Vec3f(1, 1, 1))

command = `go run ./cmd/catnip -d spotify -r 122880 -n 2048 -sas 5 -sf 45 -i -nw -nwb 50`

function run()
  open(command, "r", stdout) do io
    count = 0
    while count < numSets + 2
      count += 1

      line = readline(io)
      line = rstrip(lstrip(line))
      elms = split(line, " ")
      nums = map(x -> parse(Float64, x), elms)
      pushfirst!(data, reverse(nums))
    end

    updateZ()

    limits!(ax1, 0, sets[], 0, bars[], 0, 100)

    meshscatter!(ax1, x, y, zZ; marker=rectMesh, color=staticColorMap,
      markersize=mSize, colormap=:summer,
      shading=true)

    while !eof(io)
      line = readline(io)
      line = rstrip(lstrip(line))
      elms = split(line, " ")
      nums = map(x -> parse(Float64, x), elms)
      nums = reverse(nums)

      println(nums)

      pushfirst!(data, nums)

      updateZ()
    end
  end
end


display(fig)

run()
