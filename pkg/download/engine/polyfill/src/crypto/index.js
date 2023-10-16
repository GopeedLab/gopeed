globalThis.crypto = {
    getRandomValues(arr) {
        for (let i = 0, len = arr.length; i < len; i++) {
            arr[i] = Math.floor(Math.random() * 256);
        }
        return arr;
    },
    randomUUID() {
        return '10000000-1000-4000-8000-100000000000'.replace(/[018]/g, (c) => {
            return (c ^ (this.getRandomValues(new Uint8Array(1))[0] & (15 >> (c / 4)))).toString(16)
        })
    }
}