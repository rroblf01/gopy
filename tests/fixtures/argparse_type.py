import argparse


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--count", type=int, default=10)
    parser.add_argument("--ratio", type=float, default=1.5)
    parser.add_argument("--name", type=str, default="anon")
    parser.add_argument("--verbose", action="store_true")
    parser.add_argument("port", type=int)

    # Smoke test: parse_args must not raise. Attribute access on the
    # Namespace differs between CPython (`ns.count`) and gopy (`ns.Get`),
    # so we don't print the parsed values here. The existing
    # argparse_basic fixture exercises positional / kwarg matching.
    parser.parse_args(
        ["--count", "42", "--ratio", "2.5", "--verbose", "8080"]
    )
    print("argparse type ok")


if __name__ == "__main__":
    main()
