import sys


def main() -> None:
    sys.setrecursionlimit(2000)
    sys.setswitchinterval(0.005)
    sys.audit("event_name", "data")
    sys.settrace(None)
    sys.setprofile(None)
    # All hooks accepted no-op. Just confirm it didn't crash.
    print("ok")


if __name__ == "__main__":
    main()
