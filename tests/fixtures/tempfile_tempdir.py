import os
from tempfile import TemporaryDirectory


def main() -> None:
    saved: str = ""
    with TemporaryDirectory(prefix="gopy_td_") as d:
        f = os.path.join(d, "data.txt")
        with open(f, "w") as fh:
            fh.write("inside")
        with open(f) as fh:
            print(fh.read())
        saved = d
        print(os.path.isdir(d))

    # Directory removed after the `with` block exits.
    print(os.path.isdir(saved))


if __name__ == "__main__":
    main()
