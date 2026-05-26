import argparse


def main() -> None:
    p = argparse.ArgumentParser()
    g = p.add_mutually_exclusive_group()
    g.add_argument("--foo", action="store_true")
    g.add_argument("--bar", action="store_true")

    p.parse_args(["--foo"])
    print("ok")

    try:
        p.parse_args(["--foo", "--bar"])
        print("not rejected")
    except SystemExit:
        print("rejected")
    except Exception:
        print("rejected")


if __name__ == "__main__":
    main()
