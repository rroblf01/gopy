import asyncio


class CM:
    def __init__(self, name: str) -> None:
        self.name: str = name

    async def __aenter__(self) -> str:
        print("enter", self.name)
        return self.name

    async def __aexit__(self, exc_type: object, exc: object, tb: object) -> None:
        print("exit", self.name)


async def fake_fetch(n: int) -> int:
    return n * 2


async def sum_doubles(items: list[int]) -> int:
    total: int = 0
    for v in items:
        total += await fake_fetch(v)
    return total


async def use_cm() -> None:
    async with CM("ctx") as name:
        print("inside", name)


async def main() -> None:
    print(await sum_doubles([1, 2, 3, 4]))
    await use_cm()


if __name__ == "__main__":
    asyncio.run(main())
