import zipfile
import os
import tempfile


def main() -> None:
    d = tempfile.mkdtemp()
    os.chdir(d)
    with open("a.txt", "w") as fh:
        fh.write("Apple")
    with zipfile.ZipFile("out.zip", "w") as zf:
        zf.write("a.txt")
        zf.writestr("inline.txt", "Inline body")

    with zipfile.ZipFile("out.zip") as zf:
        names = zf.namelist()
        out: list[str] = []
        for n in names:
            out.append(str(n))
        out.sort()
        print(",".join(out))


if __name__ == "__main__":
    main()
