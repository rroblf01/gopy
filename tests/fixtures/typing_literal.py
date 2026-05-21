from typing import Literal


def role(name: Literal["admin", "user", "guest"]) -> int:
    if name == "admin":
        return 0
    if name == "user":
        return 1
    return 2


def main() -> None:
    print(role("admin"))
    print(role("user"))
    print(role("guest"))


if __name__ == "__main__":
    main()
