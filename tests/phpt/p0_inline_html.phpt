--TEST--
P0 inline HTML source mode
--FILE--
Hello <?php echo 'PHP'; ?>
<?= 'short' ?>
Done
--EXPECT--
Hello PHP
short
Done
