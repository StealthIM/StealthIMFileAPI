import typer
import os
from . import hash as hash_func
from . import upload as upload_func
from . import download as download_func
from . import usage as reload_func
import sys
from rich.console import Console
import asyncio
import time
import shutil

console = Console()
app = typer.Typer()

server_host = "127.0.0.1"
server_port = 50053
blocksize = 2048*1024


@app.callback()
def global_cmd(
        server_host_input: str = typer.Option(
            "127.0.0.1:50053",
            "--server",
            "-S",
            show_default=False,
            help="Server address"
        ),
        blocksize_input: int = typer.Option(
            2048,
            "--blocksize",
            "-B",
            help="Set block size(KiB)",
            is_flag=True
        )):
    global server_host, server_port, verbose, blocksize
    server_host = server_host_input.split(":")[0]
    server_port = int(server_host_input.split(":")[1])
    blocksize = blocksize_input*1024


@app.command(name="upload", help="Upload File")
def upload(path: str):
    if not os.path.exists(path):
        console.print("[bold red]File Not Found: [/bold red]"+path)
        raise typer.Exit(1)
    console.print(
        f"[blue][bold]filesize[/bold]: {os.stat(path).st_size}[/blue]")
    upload_time = time.time()
    console.print("[green]calculating hash...[/green]", end="")
    filehash = hash_func.getfilehash(path, block_size=blocksize)
    sys.stdout.write("\r"+" "*console.width+"\r")
    console.print(
        f"[green][bold]hash[/bold]: [underline]{filehash}[/underline][/green]")
    asyncio.run(upload_func.upload(
        server_host, server_port, path, blocksize, filehash))
    console.print(
        f"[blue][bold]Upload Complete![/bold][/blue]")
    console.print(
        f"Used time: {time.time()-upload_time:.02} s")


@app.command(name="download", help="Download File")
def download(hash: str, path: str):
    path = os.path.abspath(path)
    if not os.path.exists(os.path.dirname(os.path.abspath(path))):
        console.print("[bold red]File Not Found: [/bold red]"+path)
        raise typer.Exit(1)
    if os.path.exists(path):
        console.print("[bold red]File Already Exists: [/bold red]"+path)
        raise typer.Exit(1)
    if os.path.exists(path+".stimfstmp"):
        console.print("[yellow]File Already Exists: [/yellow]"+path)
    else:
        os.mkdir(path+".stimfstmp")
    blocknum = asyncio.run(  # type: ignore
        download_func.download(  # type: ignore
            server_host, server_port, path, blocksize, hash))  # type: ignore
    console.print(
        "[blue bold]Merge file...[/blue bold]", end="")

    with open(path, "wb") as f:
        for i in range(blocknum):
            with open(path+".stimfstmp"+os.sep+str(i), "rb") as f2:
                f.write(f2.read())

    sys.stdout.write("\r"+" "*console.width+"\r")
    console.print(
        "[blue bold]Clean file...[/blue bold]", end="")
    shutil.rmtree(path+".stimfstmp")
    sys.stdout.write("\r"+" "*console.width+"\r")
    console.print(
        "[blue bold]Check file hash...[/blue bold]", end="")
    if hash != hash_func.getfilehash(path, block_size=blocksize):
        console.print(
            "\n[red bold]Check hash fault! [/red bold]")
        os.remove(path)
        raise typer.Exit(2)
    sys.stdout.write("\r"+" "*console.width+"\r")
    console.print(
        "[green bold]Download Success![/green bold]")


@app.command(name="reload", help="Reload Config")
def reload():
    asyncio.run(reload_func.reload(server_host, server_port))


@app.command(name="usage", help="Get Node Usage")
def usage():
    asyncio.run(reload_func.usage(server_host, server_port))
