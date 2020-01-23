remote_exec "nomad_jobs" {
    target = "container.tools"
    cmd = "nomad"
    args = ["run", "/files/example.nomad"]
}