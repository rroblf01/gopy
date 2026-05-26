import shelve
import os
import tempfile


def main() -> None:
    d = tempfile.mkdtemp()
    path = os.path.join(d, "store")
    sh = shelve.open(path)
    sh["a"] = 1
    sh["b"] = 2
    sh["c"] = 3
    print("a" in sh)
    print("z" in sh)
    print(len(sh))
    del sh["b"]
    print(len(sh))
    print("b" in sh)
    val = sh["a"]
    print(val)
    sh.close()


if __name__ == "__main__":
    main()
