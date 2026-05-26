import os
import tempfile


def main() -> None:
    d = tempfile.mkdtemp()
    p = os.path.join(d, "f.txt")
    with open(p, "w") as fh:
        fh.write("x")
    os.utime(p, (1000000.0, 1000000.0))
    print(int(os.path.getmtime(p)))


if __name__ == "__main__":
    main()
