--TEST--
p0 exit code
--FILE--
<?php
exit(3);
--EXPECT--

--EXIT_CODE--
3
