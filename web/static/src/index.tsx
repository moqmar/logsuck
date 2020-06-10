import { h, render } from "preact";
import { HomeComponent } from "./pages/home";
import { search, startJob, pollJob, getResults } from "./api/v1";

async function main() {
    const appRoot = document.getElementById("app");
    if (!appRoot) {
        throw new Error("No element with id 'app' found!");
    }
    render(<HomeComponent searchApi={search} startJob={startJob} pollJob={pollJob} getResults={getResults} />, appRoot);
}

main();
