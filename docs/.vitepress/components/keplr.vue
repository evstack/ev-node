<template>
  <button @click.native="addNetwork"><img src="./keplr.png" alt="" width="20" height="20" /></button>
</template>

<script>
export default {
  name: 'app',
  methods: {
    async addNetwork() {
      try {
        const settings = await import(`./rosm.json`);
        console.log("got back settings", settings);
        try {
          await window.keplr.enable(settings.chainId);
          alert(settings.chainId + " already added");
        } catch (e) {
          console.log(
            "Unable to connect to wallet natively, so trying experimental chain"
          );
          try {
            await window.keplr.experimentalSuggestChain(settings);
            await window.keplr.enable(settings.chainId);
          } catch (e2) {
            console.log(
              "and yet there is a problem in trying to do that too",
              e2
            );
          }
        }
      } catch (error) {
        if (error instanceof SyntaxError) {
          alert(
            "There was a syntax error. Please correct it and try again: " +
              error.message
          );
        } else {
          throw error;
        }
      }
    }
  },
}
</script>