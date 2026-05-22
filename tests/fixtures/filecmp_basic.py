import filecmp
import os
import tempfile


def main() -> None:
    d = tempfile.mkdtemp()
    a = os.path.join(d, "a.txt")
    b = os.path.join(d, "b.txt")
    c = os.path.join(d, "c.txt")
    with open(a, "w") as f:
        f.write("hello")
    with open(b, "w") as f:
        f.write("hello")
    with open(c, "w") as f:
        f.write("world")
    print(filecmp.cmp(a, b))
    print(filecmp.cmp(a, c))


if __name__ == "__main__":
    main()
