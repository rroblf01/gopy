import argparse


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--name")
    parser.add_argument("--count")
    parser.add_argument("city")
    # We only assert the call chain works end-to-end in both CPython and
    # gopy; attribute access on the Namespace differs (`ns.name` in CPython
    # vs `ns.Get("name")` in gopy), so we don't compare the parsed values
    # here. The smoke check is enough to detect regressions in lowering.
    parser.parse_args(["--name", "ana", "--count", "3", "madrid"])
    print("argparse ok")


if __name__ == "__main__":
    main()
