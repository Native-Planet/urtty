import adapter from '@sveltejs/adapter-static';

const makeKit = () => {
  let kit = {
    kit: {
      adapter: adapter({
        pages: 'build',
        assets: 'build',
        fallback: 'index.html',
        precompress: false,
        strict: true
      }),
    }
  }
	kit.kit["paths"] = { 
		base: '/apps/urtty',
	}
  return kit
}
const config = makeKit();
export default config