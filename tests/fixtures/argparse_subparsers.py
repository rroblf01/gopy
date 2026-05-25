import argparse


def main() -> None:
    p = argparse.ArgumentParser()
    subs = p.add_subparsers(dest="cmd")
    add = subs.add_parser("add")
    add.add_argument("--n", type=int, default=1)
    rm = subs.add_parser("rm")
    rm.add_argument("--name", default="")
    # Smoke test: just ensure the parser builds and parses without
    # error. Both runtimes accept the call shape; namespace access
    # differs (CPython uses attribute, gopy uses .get()) so we don't
    # print the namespace here.
    p.parse_args(["add", "--n", "7"])
    print("ok")


if __name__ == "__main__":
    main()
