<script lang="ts">
    // This component intentionally never resolves to trigger the boundary's pending state
    const never = new Promise<never>(() => {});
</script>

<div>{await never}</div>
