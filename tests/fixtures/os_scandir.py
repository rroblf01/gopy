import os
import tempfile


def main() -> None:
    base = tempfile.mkdtemp(prefix="gopy_sd_")
    os.makedirs(os.path.join(base, "sub"))
    with open(os.path.join(base, "a.txt"), "w") as fh:
        fh.write("a")
    with open(os.path.join(base, "b.log"), "w") as fh:
        fh.write("b")

    names: list[str] = []
    for entry in os.scandir(base):
        names.append(entry.name)
    names.sort()
    for n in names:
        print(n)


if __name__ == "__main__":
    main()
