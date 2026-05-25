import tarfile
import os
import tempfile
import subprocess


def main() -> None:
    d = tempfile.mkdtemp()
    os.chdir(d)
    with open("hello.txt", "w") as fh:
        fh.write("hi\n")
    subprocess.run(["tar", "-cf", "demo.tar", "hello.txt"])
    tar_path = "demo.tar"

    with tarfile.open(tar_path) as tf:
        names = tf.getnames()
        out: list[str] = []
        for n in names:
            out.append(str(n))
        out.sort()
        print(",".join(out))

    with tarfile.open(tar_path) as tf:
        tf.extractall("out")

    body = ""
    with open("out/hello.txt", "r") as fh:
        body = fh.read()
    print(body.strip())


if __name__ == "__main__":
    main()
