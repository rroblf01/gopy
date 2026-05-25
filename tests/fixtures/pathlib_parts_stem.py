from pathlib import Path


def main() -> None:
    p = Path("/a/b/c/file.txt")

    print(p.with_stem("renamed"))
    print(p.with_suffix(".log"))

    # parts returns the segments; CPython yields a tuple, gopy a list.
    # Cast to list in CPython for portable iteration.
    for seg in list(p.parts):
        print(seg)

    # parents is iterable; lengths differ across OSes if mixed slashes,
    # so just print the first one (= immediate parent).
    ps = list(p.parents)
    print(str(ps[0]))


if __name__ == "__main__":
    main()
