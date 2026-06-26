#!/usr/bin/env pwsh
# Generate protobuf Go code
param(
    [string]$ProtoDir = "pkg/proto"
)

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RootDir = Split-Path -Parent $ScriptDir
$ProtoRoot = Join-Path $RootDir $ProtoDir
$LocalProtoc = ".tools/protoc-35.1/bin/protoc.exe"
$LocalInclude = ".tools/protoc-35.1/include"
$LocalProtocGenGo = ".gopath/bin/protoc-gen-go.exe"
$LocalProtocGenGoGRPC = ".gopath/bin/protoc-gen-go-grpc.exe"

$Protoc = "protoc"
if (Test-Path (Join-Path $RootDir $LocalProtoc)) {
    $Protoc = $LocalProtoc
}

if (-not (Test-Path $ProtoRoot)) {
    throw "Proto directory not found: $ProtoRoot"
}

Write-Output "Generating protobuf code..."
Write-Output "  protoc: $Protoc"

$protos = Get-ChildItem -Path $ProtoRoot -Recurse -Filter "*.proto"
Push-Location $RootDir
try {
    foreach ($proto in $protos) {
        $relativeProto = Resolve-Path -Path $proto.FullName -Relative
        $relativeProto = $relativeProto.TrimStart(".\").Replace("\", "/")
        Write-Output "  Compiling: $relativeProto"

        $args = @(
            "--proto_path=.",
            "--proto_path=$LocalInclude",
            "--go_out=.",
            "--go_opt=paths=source_relative",
            "--go-grpc_out=.",
            "--go-grpc_opt=paths=source_relative"
        )
        if (Test-Path $LocalProtocGenGo) {
            $args += "--plugin=protoc-gen-go=$LocalProtocGenGo"
        }
        if (Test-Path $LocalProtocGenGoGRPC) {
            $args += "--plugin=protoc-gen-go-grpc=$LocalProtocGenGoGRPC"
        }
        $args += $relativeProto

        & $Protoc @args
        if ($LASTEXITCODE -ne 0) {
            throw "protoc failed for $relativeProto with exit code $LASTEXITCODE"
        }
    }
}
finally {
    Pop-Location
}

Write-Output "Protobuf code generation complete!"
