import tarfile
import os
import tempfile


def main() -> None:
    d = tempfile.mkdtemp()
    os.chdir(d)
    with open("alpha.txt", "w") as fh:
        fh.write("A")
    with open("beta.txt", "w") as fh:
        fh.write("B")

    with tarfile.open("bundle.tar", "w") as tf:
        tf.add("alpha.txt")
        tf.add("beta.txt")

    with tarfile.open("bundle.tar") as tf:
        names = tf.getnames()
        out: list[str] = []
        for n in names:
            s: str = str(n)
            if s == "." or s == "":
                continue
            if s.startswith("./"):
                s = s[2:]
            out.append(s)
        out.sort()
        print(",".join(out))


if __name__ == "__main__":
    main()
