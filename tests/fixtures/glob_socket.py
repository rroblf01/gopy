import glob
import os
import socket
import tempfile


def main() -> None:
    d = tempfile.mkdtemp("gopy_glob_")
    with open(d + "/a.txt", "w") as fh:
        fh.write("x")
    with open(d + "/b.txt", "w") as fh:
        fh.write("y")
    with open(d + "/c.log", "w") as fh:
        fh.write("z")
    found: list[str] = glob.glob(d + "/*.txt")
    found.sort()
    for p in found:
        print(os.path.basename(p))
    h = socket.gethostname()
    print(len(h) > 0)


if __name__ == "__main__":
    main()
