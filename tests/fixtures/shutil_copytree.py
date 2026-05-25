import os
import shutil
import tempfile


def main() -> None:
    src = tempfile.mkdtemp(prefix="gopy_src_")
    dst_base = tempfile.mkdtemp(prefix="gopy_dst_")
    dst = os.path.join(dst_base, "tree")

    os.makedirs(os.path.join(src, "a", "b"))
    with open(os.path.join(src, "top.txt"), "w") as fh:
        fh.write("top")
    with open(os.path.join(src, "a", "leaf.txt"), "w") as fh:
        fh.write("leaf")

    shutil.copytree(src, dst)

    # Probe a known leaf and a directory entry.
    print(os.path.isfile(os.path.join(dst, "top.txt")))
    print(os.path.isdir(os.path.join(dst, "a")))
    print(os.path.isfile(os.path.join(dst, "a", "leaf.txt")))

    with open(os.path.join(dst, "a", "leaf.txt")) as fh:
        print(fh.read())


if __name__ == "__main__":
    main()
