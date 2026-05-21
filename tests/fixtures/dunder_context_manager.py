from typing import Any


class Trace:
    def __init__(self, label: str) -> None:
        self.label = label
        self.entered = False

    def __enter__(self) -> "Trace":
        print(f"enter {self.label}")
        self.entered = True
        return self

    def __exit__(self, exc_type: Any, exc_val: Any, exc_tb: Any) -> None:
        print(f"exit {self.label}")
        self.entered = False


def main() -> None:
    with Trace("outer") as t:
        print(t.entered)
        print(t.label)
        with Trace("inner") as u:
            print(u.label)
    print("done")


if __name__ == "__main__":
    main()
