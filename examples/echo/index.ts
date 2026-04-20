import { attachment, custom, Spectrum, text } from "spectrum-ts";
import { tuichat } from "tuichat";

const app = await Spectrum({
  providers: [
    tuichat.config({
      commands: [
        { name: "/attach", description: "send a demo attachment" },
        { name: "/custom", description: "send a demo custom payload" },
        { name: "/multi", description: "send three messages in sequence" },
      ],
    }),
  ],
});

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

for await (const [space, message] of app.messages) {
  if (message.content.type !== "text") continue;

  const incoming = message.content.text;

  if (incoming === "/attach") {
    await space.responding(async () => {
      await sleep(300);
      const bytes = Buffer.from("demo attachment contents — hello world\n");
      await space.send(attachment(bytes, { name: "note.txt" }));
    });
    continue;
  }

  if (incoming === "/custom") {
    await space.responding(async () => {
      await sleep(300);
      await space.send(custom({ kind: "demo", at: new Date().toISOString(), n: 42 }));
    });
    continue;
  }

  if (incoming === "/multi") {
    await space.responding(async () => {
      await sleep(300);
      await space.send(text("first"));
      await space.send(text("second"));
      await space.send(text("third"));
    });
    continue;
  }

  await space.responding(async () => {
    await sleep(400);
    await space.send(`echo: ${incoming}`);
  });
}
