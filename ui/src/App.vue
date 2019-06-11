<template>
  <div>
    <h1 class="hero__headline">Supertar</h1>
    <div id="app">
      <Browser/>
    </div>
    <footer class="footer">
      Supertar on
      <a href="https://www.github.com/marcboeker/supertar" target="_blank">Github</a> | UI based on work by
      <a href="https://codepen.io/l4ci/pen/BYNGxw" target="_blank">Volker Otto</a>
    </footer>
  </div>
</template>

<script>
import Browser from "./components/Browser.vue";

export default {
  name: "app",
  components: {
    Browser
  }
};
</script>

<style lang="scss">
#app {
  position: relative;
  display: flex;
  margin: 0 auto;
  max-width: 1000px;
  min-height: 300px;
  background: #fff;
  overflow: hidden;

  @media (min-width: 900px) {
    min-height: 50vh;
    box-shadow: 0 0 20px rgba(#787878, 0.1);
    border-radius: 5px;
  }
}

body {
  background: #efefef;
  font-size: 100%;
  line-height: 1.5;
  font-size: 0.875rem;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto,
    Oxygen-Sans, Ubuntu, Cantarell, "Helvetica Neue", Helvetica, Arial,
    sans-serif, "Apple Color Emoji", "Segoe UI Emoji", "Segoe UI Symbol";
  color: #787878;

  @media (min-width: 900px) {
    padding: 1rem;
  }
}

a {
  text-decoration: none;
  color: inherit;
}

html {
  box-sizing: border-box;
}

*,
*::before,
*::after {
  box-sizing: inherit;
}

.hero__headline {
  text-align: center;
  font-weight: 500;
}

.footer {
  text-align: center;
  font-size: 0.75rem;
  margin-top: 1rem;
  color: #bcbcbc;
}
</style>
