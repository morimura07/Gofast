<script lang="ts">
    async function fail(): Promise<never> {
        throw new Error("Boom");
    }
</script>

<div>{await fail()}</div>
