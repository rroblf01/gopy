from urllib.robotparser import RobotFileParser


def main() -> None:
    # Parser without read() returns True for any URL (no rules loaded).
    rp = RobotFileParser()
    rp.set_url("https://example.invalid/robots.txt")
    print(rp.can_fetch("ua", "/foo"))


if __name__ == "__main__":
    main()
