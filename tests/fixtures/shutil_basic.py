import os
import shutil
import tempfile


def main() -> None:
    d = tempfile.mkdtemp("gopy_shutil_")
    print(os.path.isdir(d))
    src = d + "/a.txt"
    dst = d + "/b.txt"
    with open(src, "w") as fh:
        fh.write("hello")
    shutil.copy(src, dst)
    print(os.path.isfile(dst))
    moved = d + "/c.txt"
    shutil.move(dst, moved)
    print(os.path.isfile(dst))
    print(os.path.isfile(moved))
    shutil.rmtree(d)
    print(os.path.isdir(d))


if __name__ == "__main__":
    main()
