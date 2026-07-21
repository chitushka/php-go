--TEST--
P0 heredoc nowdoc and backticks
--FILE--
<?php
$name = 'PHP';

$heredoc = <<<TXT
Hello $name
TXT;

$nowdoc = <<<'RAW'
Raw $name
RAW;

$cmd = 'BT';
echo $heredoc;
echo $nowdoc;
echo `echo $cmd`;
?>
--EXPECT--
Hello PHPRaw $nameBT
