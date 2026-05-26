import shutil
import os
import tempfile
import tarfile


def main() -> None:
    d = tempfile.mkdtemp()
    src_dir = os.path.join(d, "src")
    os.mkdir(src_dir)
    with open(os.path.join(src_dir, "alpha.txt"), "w") as fh:
        fh.write("A")
    with open(os.path.join(src_dir, "beta.txt"), "w") as fh:
        fh.write("B")
    base = os.path.join(d, "bundle")
    out = shutil.make_archive(base, "tar", src_dir)
    print(out.endswith(".tar"))
    with tarfile.open(out) as tf:
        names = tf.getnames()
        collected: list[str] = []
        for n in names:
            s: str = str(n)
            if s == "." or s == "":
                continue
            if s.startswith("./"):
                s = s[2:]
            collected.append(s)
        collected.sort()
        print(",".join(collected))


if __name__ == "__main__":
    main()
