1. Create `.env` at the repo root with the following content:

  ``` 
   WEBUI_PORT=3000
   STRIPPER_PORT=8080
   STRIPPER_HOST_PORT=8081 
   ```

2. Start both services (run from the root of the repository):
  ``` 
    docker compose up --build
  ``` 
3. Try it out in your browser:

- Open the web UI: `http://localhost:3000`
- Upload `.jpg`/`.jpeg` 
- Click "Clean Metadata" -> you'll get a `cleaned.jpg` download with EXIF metadata removed.