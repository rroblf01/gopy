import shutil
import os
import tempfile


def main() -> None:
    d = tempfile.mkdtemp()
    src = os.path.join(d, "src")
    os.mkdir(src)
    with open(os.path.join(src, "a.txt"), "w") as fh:
        fh.write("A")
    base = os.path.join(d, "bundle")
    out = shutil.make_archive(base, "tar", src)
    dest = os.path.join(d, "out")
    shutil.unpack_archive(out, dest)
    extracted = os.path.join(dest, "a.txt")
    body: str = ""
    with open(extracted, "r") as fh:
        body = fh.read()
    print(body.strip())


if __name__ == "__main__":
    main()
