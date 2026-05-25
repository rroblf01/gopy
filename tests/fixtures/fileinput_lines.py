import fileinput
import os
import tempfile


def main() -> None:
    d = tempfile.mkdtemp()
    a = os.path.join(d, "a.txt")
    b = os.path.join(d, "b.txt")
    with open(a, "w") as fh:
        fh.write("alpha\nbeta\n")
    with open(b, "w") as fh:
        fh.write("gamma\n")
    total = 0
    lines = fileinput.input([a, b])
    for raw in lines:
        total += 1
        s: str = raw
        print(s.strip())
    print(total)


if __name__ == "__main__":
    main()
